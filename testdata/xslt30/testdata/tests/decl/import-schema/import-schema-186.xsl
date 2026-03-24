<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:a="http://ns.example.com/sch002"
                xmlns:b="http://ns.example.com/sch004"
                exclude-result-prefixes="a b"
                version="2.0">

  <!-- Purpose: Test where two different schemas with different namespaces are imported, 
				one in the primary and one in the imported stylesheet. Show that datatypes 
				from both schemas can be used in both stylesheets.-->

   <xsl:import href="import-schema-186a.xsl"/>
   <xsl:import-schema namespace="http://ns.example.com/sch002"
                      schema-location="testSchemaInline.xsd"/>


   <xsl:variable name="var1" as="element()">
      <a:complexTest xsl:type="a:complexTestType" a:type="333">
         <subject>math</subject>
         <size>50</size>
      </a:complexTest>
   </xsl:variable>

   <xsl:variable name="var2" as="element()">
	     <b:complexMixedUserElem age="44" xsl:type="b:mixedPersonType">
         <description>Mr <name>Peter</name> has brown hair</description>
         <company>He works at a <name>Notorios</name> company</company>
      </b:complexMixedUserElem>
   </xsl:variable>

   <xsl:template match="/doc">
      <xsl:text>
</xsl:text>
      <out>
         <xsl:text>
</xsl:text>
         <primary1>
            <xsl:value-of select="$var1 instance of schema-element(a:complexTest)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var1/@a:type instance of schema-attribute(a:type)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$var1"/>
         </primary1>
         <xsl:text>
</xsl:text>
         <primary2>
            <xsl:value-of select="$var2 instance of element(*, b:mixedPersonType)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var2 instance of schema-element(b:complexMixedUserElem)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$var2/*[1]"/>
         </primary2>
         <xsl:call-template name="imported"/>
      </out>
   </xsl:template>



</xsl:stylesheet>
