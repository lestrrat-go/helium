<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/subgroup"
   xmlns:pre="http://www.example.com/ns/match/subgroup" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my xs pre" version="2.0">
   <!-- Purpose: Test of template matching with schema-element($name) with an explicitly created and typed element node without a parent. 
  				Show all cases where a node will match the pattern -->

   <t:import-schema namespace="http://www.example.com/ns/match/subgroup"
      schema-location="subgroupTypeSchemaMatch.xsd"/>

   <t:variable name="main" as="element()">
      <my:elem-anyURI t:type="xs:anyURI">http://ns.example.com</my:elem-anyURI>
   </t:variable>

   <t:variable name="v" as="element()">
      <my:elem-anyURI-sub t:type="xs:anyURI">http://www.example.com/test</my:elem-anyURI-sub>
   </t:variable>

   <t:variable name="v1" as="element()">
      <my:elem-anyURI-sub1 t:type="my:derivedURI">http://test1.org</my:elem-anyURI-sub1>
   </t:variable>

   <t:variable name="v2" as="element()">
      <my:elem-anyURI-sub2 t:type="my:derivedURI2">http://test2.org</my:elem-anyURI-sub2>
   </t:variable>

   <t:variable name="v3" as="element()">
      <my:elem-anyURI-sub3 t:type="my:derivedURI3">http://test3.org</my:elem-anyURI-sub3>
   </t:variable>

   <t:template match="/">
      <out>
         <t:apply-templates select="$main"/>
         <t:apply-templates select="$v"/>
         <t:apply-templates select="$v1"/>
         <t:apply-templates select="$v2"/>
         <t:apply-templates select="$v3"/>
      </out>
   </t:template>

   <t:template match="schema-element(pre:elem-anyURI)">
      <t:element name="A{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>
