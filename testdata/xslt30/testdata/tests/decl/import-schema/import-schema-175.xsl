<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:foo="http://ns.example.com/sch004" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="foo xs" version="2.0">

   <!-- Purpose: Test of xsl:import-schema with @namespace and @schema-location importing a more complex schema.Verify 
  				that top-level element and attribute declarations and defined types that are in  @namespace are available 
  				for use in the stylesheet. -->

   <xsl:import-schema namespace="http://ns.example.com/sch004" schema-location="schema004.xsd"/>

   <xsl:template match="/foo:userNode">
      <out>
         <E1>
            <xsl:value-of select="data(foo:simpleUserElem) instance of foo:partNumberType"/>
         </E1>
         <E2>
            <xsl:value-of
               select="foo:complexMixedUserElem instance of element(*, foo:mixedPersonType)"/>
         </E2>
         <E3>
            <xsl:value-of
               select="foo:complexUserElem instance of schema-element(foo:complexUserElem)"/>
         </E3>
         <A1>
            <xsl:value-of select="data(@foo:specialPart) instance of foo:partNumberType"/>
         </A1>
         <A2>
            <xsl:value-of select="@foo:listParts instance of attribute(*, foo:myListType)"/>
         </A2>
         <A3>
            <xsl:value-of select="@foo:specialPart instance of schema-attribute(foo:specialPart)"/>
         </A3>
      </out>
   </xsl:template>

</xsl:stylesheet>
