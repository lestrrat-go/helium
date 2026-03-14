<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://ns.example.com/strip-type" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" input-type-annotations="strip" version="2.0">
   <!-- Purpose: Show that type annotations in temporary trees or element and attribute nodes  constructed in the 
  				stylesheet are not affected by @input-type-annotations="strip".-->

   <xslt:import-schema namespace="http://ns.example.com/strip-type"
      schema-location="varietyOfTypes.xsd"/>

   <xslt:variable name="var">
      <complex-derived age="21" xslt:type="my:personType">
         <firstName>John</firstName>
         <familyName>Brigs</familyName>
      </complex-derived>
   </xslt:variable>

   <xslt:variable name="var1" as="element(*, my:personType)">
      <xslt:element name="complex-derived" type="my:personType">
         <xslt:attribute name="age">21</xslt:attribute>
         <xslt:element name="firstName">John</xslt:element>
         <xslt:element name="familyName">Brigs</xslt:element>
      </xslt:element>
   </xslt:variable>

   <xslt:variable name="var2" as="attribute(*, xs:integer)">
      <xslt:attribute name="age" type="xs:integer">21</xslt:attribute>
   </xslt:variable>

   <xslt:template match="/doc">

      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

         <out1>
            <xslt:value-of select="$var/complex-derived instance of element(*, my:personType)"/>
            <xslt:value-of select="$var/complex-derived instance of element(*, xs:untyped)"/>
            <xslt:value-of select="$var/complex-derived/@age instance of attribute(*, xs:integer)"/>
            <xslt:value-of
               select="$var/complex-derived/@age instance of attribute(*, xs:untypedAtomic)"/>
         </out1>

         <out2>
            <xslt:value-of select="$var1 instance of element(*, my:personType)"/>
            <xslt:value-of select="$var1 instance of element(*, xs:untyped)"/>
            <xslt:value-of select="$var1/@age instance of attribute(*, xs:integer)"/>
            <xslt:value-of select="$var1/@age instance of attribute(*, xs:untypedAtomic)"/>
         </out2>

         <out3 var2="{$var2}">
            <xslt:value-of select="$var2 instance of attribute(*, xs:integer)"/>
            <xslt:value-of select="$var2 instance of attribute(*, xs:untypedAtomic)"/>
         </out3>

      </output>
   </xslt:template>
</xslt:transform>
