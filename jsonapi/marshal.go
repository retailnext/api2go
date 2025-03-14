package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// RelationshipType specifies the type of a relationship.
type RelationshipType int

// The available relationship types.
//
// Note: DefaultRelationship guesses the relationship type based on the
// pluralization of the reference name.
const (
	DefaultRelationship RelationshipType = iota
	ToOneRelationship
	ToManyRelationship
)

// The MarshalIdentifier interface is necessary to give an element a unique ID.
//
// Note: The implementation of this interface is mandatory.
type MarshalIdentifier interface {
	GetID() Identifier
}

type Identifier struct {
	ID   string `json:"id"`
	LID  string `json:"lid,omitempty"`
	Name string `json:"type"`
}

// ReferenceID contains all necessary information in order to reference another
// struct in JSON API.
type ReferenceID struct {
	ID           string
	LID          string
	Type         string
	Name         string
	Relationship RelationshipType
}

// A Reference information about possible references of a struct.
//
// Note: If IsNotLoaded is set to true, the `data` field will be omitted and only
// the `links` object will be generated. You should do this if there are some
// references, but you do not want to load them. Otherwise, if IsNotLoaded is
// false and GetReferencedIDs() returns no IDs for this reference name, an
// empty `data` field will be added which means that there are no references.
type Reference struct {
	Type         string
	Name         string
	IsNotLoaded  bool
	Relationship RelationshipType
}

// The MarshalReferences interface must be implemented if the struct to be
// serialized has relationships.
type MarshalReferences interface {
	GetReferences() []Reference
}

// The MarshalLinkedRelations interface must be implemented if there are
// reference ids that should be included in the document.
type MarshalLinkedRelations interface {
	MarshalReferences
	MarshalIdentifier
	GetReferencedIDs() []ReferenceID
}

// The MarshalIncludedRelations interface must be implemented if referenced
// structs should be included in the document.
type MarshalIncludedRelations interface {
	MarshalReferences
	MarshalIdentifier
	GetReferencedStructs() []MarshalIdentifier
}

// The MarshalCustomLinks interface can be implemented if the struct should
// want any custom links.
type MarshalCustomLinks interface {
	MarshalIdentifier
	GetCustomLinks(string) Links
}

// The MarshalMeta interface can be implemented if the struct should
// want any meta.
type MarshalMeta interface {
	MarshalIdentifier
	Meta() Meta
}

// The MarshalCustomRelationshipMeta interface can be implemented if the struct should
// want a custom meta in a relationship.
type MarshalCustomRelationshipMeta interface {
	MarshalIdentifier
	GetCustomMeta(string) map[string]Meta
}

// A ServerInformation implementor can be passed to MarshalWithURLs to generate
// the `self` and `related` urls inside `links`.
type ServerInformation interface {
	GetBaseURL() string
	GetPrefix() string
}

// MarshalWithURLs can be used to pass along a ServerInformation implementor.
func MarshalWithURLs(data interface{}, information ServerInformation) ([]byte, error) {
	document, err := MarshalToStruct(data, information)
	if err != nil {
		return nil, err
	}

	return json.Marshal(document)
}

// Marshal wraps data in a Document and returns its JSON encoding.
//
// Data can be a struct, a pointer to a struct or a slice of structs. All structs
// must at least implement the `MarshalIdentifier` interface.
func Marshal(data interface{}) ([]byte, error) {
	document, err := MarshalToStruct(data, nil)
	if err != nil {
		return nil, err
	}

	return json.Marshal(document)
}

// MarshalToStruct marshals an api2go compatible struct into a jsonapi Document
// structure which then can be marshaled to JSON. You only need this method if
// you want to extract or extend parts of the document. You should directly use
// Marshal to get a []byte with JSON in it.
func MarshalToStruct(data interface{}, information ServerInformation) (*Document, error) {
	if data == nil {
		return &Document{}, nil
	}

	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice:
		return marshalSlice(data, information)
	case reflect.Struct, reflect.Ptr:
		return marshalStruct(data.(MarshalIdentifier), information)
	default:
		return nil, errors.New("Marshal only accepts slice, struct or ptr types")
	}
}

func recursivelyEmbedIncludes(input []MarshalIdentifier) []MarshalIdentifier {
	var referencedStructs []MarshalIdentifier

	for _, referencedStruct := range input {
		included, ok := referencedStruct.(MarshalIncludedRelations)
		if ok {
			referencedStructs = append(referencedStructs, included.GetReferencedStructs()...)
		}
	}

	if len(referencedStructs) == 0 {
		return input
	}

	childStructs := recursivelyEmbedIncludes(referencedStructs)
	referencedStructs = append(referencedStructs, childStructs...)
	referencedStructs = append(input, referencedStructs...)

	return referencedStructs
}

func marshalSlice(data interface{}, information ServerInformation) (*Document, error) {
	result := &Document{}

	val := reflect.ValueOf(data)
	dataElements := make([]Data, val.Len())
	var referencedStructs []MarshalIdentifier

	for i := 0; i < val.Len(); i++ {
		k := val.Index(i).Interface()
		element, ok := k.(MarshalIdentifier)
		if !ok {
			return nil, errors.New("all elements within the slice must implement api2go.MarshalIdentifier")
		}

		err := marshalData(element, &dataElements[i], information)
		if err != nil {
			return nil, err
		}

		included, ok := k.(MarshalIncludedRelations)
		if ok {
			referencedStructs = append(referencedStructs, included.GetReferencedStructs()...)
		}
	}

	allReferencedStructs := recursivelyEmbedIncludes(referencedStructs)
	includedElements, err := filterDuplicates(allReferencedStructs, information)
	if err != nil {
		return nil, err
	}

	result.Data = &DataContainer{
		DataArray: dataElements,
	}

	if len(includedElements) > 0 {
		result.Included = includedElements
	}

	return result, nil
}

func filterDuplicates(input []MarshalIdentifier, information ServerInformation) ([]Data, error) {
	alreadyIncluded := map[string]map[Identifier]bool{}
	includedElements := []Data{}

	for _, referencedStruct := range input {
		structType := getStructType(referencedStruct)
		id := referencedStruct.GetID()

		if alreadyIncluded[structType] == nil {
			alreadyIncluded[structType] = make(map[Identifier]bool)
		}

		if !alreadyIncluded[structType][id] {
			var data Data
			err := marshalData(referencedStruct, &data, information)
			if err != nil {
				return nil, err
			}

			includedElements = append(includedElements, data)
			alreadyIncluded[structType][id] = true
		}
	}

	return includedElements, nil
}

func marshalData(element MarshalIdentifier, data *Data, information ServerInformation) error {
	refValue := reflect.ValueOf(element)
	if refValue.Kind() == reflect.Ptr && refValue.IsNil() {
		return errors.New("MarshalIdentifier must not be nil")
	}

	attributes, err := json.Marshal(element)
	if err != nil {
		return err
	}

	data.Attributes = attributes
	identifier := element.GetID()
	data.ID = identifier.ID
	data.LID = identifier.LID
	data.Type = getStructType(element)

	if information != nil {
		if customLinks, ok := element.(MarshalCustomLinks); ok {
			if data.Links == nil {
				data.Links = make(Links)
			}
			base := getLinkBaseURL(element, information)
			for k, v := range customLinks.GetCustomLinks(base) {
				if _, ok := data.Links[k]; !ok {
					data.Links[k] = v
				}
			}
		}
	}

	if casteMetaTarget, ok := element.(MarshalMeta); ok {
		meta := casteMetaTarget.Meta()
		data.Meta, err = json.Marshal(meta)
		if err != nil {
			return err
		}
	}

	if references, ok := element.(MarshalLinkedRelations); ok {
		data.Relationships = getStructRelationships(references, information)
	}

	return nil
}

func isToMany(relationshipType RelationshipType, name string) bool {
	if relationshipType == DefaultRelationship {
		return Pluralize(name) == name
	}

	return relationshipType == ToManyRelationship
}

func getMetaForRelation(metaSource MarshalCustomRelationshipMeta, name string, information ServerInformation) map[string]interface{} {
	meta := make(map[string]interface{})
	base := getLinkBaseURL(metaSource, information)
	if metaMap, ok := metaSource.GetCustomMeta(base)[name]; ok {
		for k, v := range metaMap {
			if _, ok := meta[k]; !ok {
				meta[k] = v
			}
		}
	}
	return meta
}

func getStructRelationships(relationer MarshalLinkedRelations, information ServerInformation) map[string]Relationship {
	referencedIDs := relationer.GetReferencedIDs()
	sortedResults := map[string][]ReferenceID{}
	relationships := map[string]Relationship{}

	for _, referenceID := range referencedIDs {
		sortedResults[referenceID.Name] = append(sortedResults[referenceID.Name], referenceID)
	}

	references := relationer.GetReferences()

	// helper map to check if all references are included to also include empty ones
	notIncludedReferences := map[string]Reference{}
	for _, reference := range references {
		notIncludedReferences[reference.Name] = reference
	}

	for name, referenceIDs := range sortedResults {
		relationships[name] = Relationship{}

		// if referenceType is plural, we need to use an array for data, otherwise it's just an object
		container := RelationshipDataContainer{}

		if isToMany(referenceIDs[0].Relationship, referenceIDs[0].Name) {
			// multiple elements in links
			container.DataArray = []Identifier{}
			for _, referenceID := range referenceIDs {
				container.DataArray = append(container.DataArray, Identifier{
					Name: referenceID.Type,
					ID:   referenceID.ID,
					LID:  referenceID.LID,
				})
			}
		} else {
			container.DataObject = &Identifier{
				Name: referenceIDs[0].Type,
				ID:   referenceIDs[0].ID,
				LID:  referenceIDs[0].LID,
			}
		}

		// set URLs if necessary
		links := getLinksForServerInformation(relationer, name, information)

		// get the custom meta for this relationship
		var meta map[string]interface{}
		if customMetaSource, ok := relationer.(MarshalCustomRelationshipMeta); ok {
			meta = getMetaForRelation(customMetaSource, name, information)
		}

		relationship := Relationship{
			Data:  &container,
			Links: links,
			Meta:  meta,
		}

		relationships[name] = relationship

		// this marks the reference as already included
		delete(notIncludedReferences, referenceIDs[0].Name)
	}

	// check for empty references
	for name, reference := range notIncludedReferences {
		container := RelationshipDataContainer{}

		// Plural empty relationships need an empty array and empty to-one need a null in the json
		if !reference.IsNotLoaded && isToMany(reference.Relationship, reference.Name) {
			container.DataArray = []Identifier{}
		}

		links := getLinksForServerInformation(relationer, name, information)

		// get the custom meta for this relationship
		var meta map[string]interface{}
		if customMetaSource, ok := relationer.(MarshalCustomRelationshipMeta); ok {
			meta = getMetaForRelation(customMetaSource, name, information)
		}

		relationship := Relationship{
			Links: links,
			Meta:  meta,
		}

		// skip relationship data completely if IsNotLoaded is set
		if !reference.IsNotLoaded {
			relationship.Data = &container
		}

		relationships[name] = relationship
	}

	return relationships
}

func getLinkBaseURL(element MarshalIdentifier, information ServerInformation) string {
	prefix := strings.Trim(information.GetBaseURL(), "/")
	namespace := strings.Trim(information.GetPrefix(), "/")

	if namespace != "" {
		prefix += "/" + namespace
	}

	return fmt.Sprintf("%s/%s/%s", prefix, getStructType(element), element.GetID().ID)
}

func getLinksForServerInformation(relationer MarshalLinkedRelations, name string, information ServerInformation) Links {
	if information == nil {
		return nil
	}

	links := make(Links)
	base := getLinkBaseURL(relationer, information)

	links["self"] = Link{Href: fmt.Sprintf("%s/relationships/%s", base, name)}
	links["related"] = Link{Href: fmt.Sprintf("%s/%s", base, name)}

	return links
}

func marshalStruct(data MarshalIdentifier, information ServerInformation) (*Document, error) {
	var contentData Data

	err := marshalData(data, &contentData, information)
	if err != nil {
		return nil, err
	}

	result := &Document{
		Data: &DataContainer{
			DataObject: &contentData,
		},
	}

	included, ok := data.(MarshalIncludedRelations)
	if ok {
		included, err := filterDuplicates(recursivelyEmbedIncludes(included.GetReferencedStructs()), information)
		if err != nil {
			return nil, err
		}

		if len(included) > 0 {
			result.Included = included
		}
	}

	return result, nil
}

func getStructType(data interface{}) string {
	identifier, ok := data.(MarshalIdentifier)
	if ok && identifier.GetID().Name != "" {
		return identifier.GetID().Name
	}

	reflectType := reflect.TypeOf(data)
	if reflectType.Kind() == reflect.Ptr {
		return Pluralize(Jsonify(reflectType.Elem().Name()))
	}

	return Pluralize(Jsonify(reflectType.Name()))
}
