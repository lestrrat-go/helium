<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/subgroup"
   xmlns:pre="http://www.example.com/ns/match/subgroup"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my pre" version="2.0">
   <!-- Purpose: Test of template matching with schema-element($name) with an explicitly constructed element 
  				node and validated with @validation=strict.  -->

   <t:import-schema namespace="http://www.example.com/ns/match/subgroup"
      schema-location="subgroupTypeSchemaMatch.xsd"/>

   <t:variable name="var" as="element()">
      <my:book t:validation="strict">
         <title>My first Book</title>
         <uri>http://www.first.example.com</uri>
         <price>99.99</price>
      </my:book>
   </t:variable>

   <t:template match="/">
      <out>
         <t:apply-templates select="$var"/>
      </out>
   </t:template>

   <t:template match="element(*)">
      <t:apply-templates select="*"/>
   </t:template>
   
   <t:template match="schema-element(pre:book)">
      <t:element name="A{position()}">
         <t:value-of select=". instance of element(*, my:book-Type)"/>
         <t:text> * </t:text>
         <t:value-of select="./title"/>
      </t:element>
   </t:template>
</t:transform>
