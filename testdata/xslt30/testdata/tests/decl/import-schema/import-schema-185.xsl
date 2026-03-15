<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:a="http://ns.example.com/sch002"
                xmlns:b="http://ns.example.com/sch004"
                xmlns:c="http://ns.example.com/sch185"
                exclude-result-prefixes="a b c"
                version="2.0">

  <!-- Purpose: Test for importing several schemas with different @namespace 
  				and @schema-location. No xs:schema content.-->
 
   <xsl:import-schema namespace="http://ns.example.com/sch002"
                      schema-location="schema002.xsd"/>
   <xsl:import-schema namespace="http://ns.example.com/sch004"
                      schema-location="variousTypesSchemaInline.xsd"/>
   <xsl:import-schema namespace="http://ns.example.com/sch185"
                      schema-location="schema185.xsd"/>

   <xsl:variable name="v1" as="element()">
      <a:complexTest xsl:type="a:complexTestType" a:type="333">
         <subject>math</subject>
         <size>50</size>
      </a:complexTest>
   </xsl:variable>

   <xsl:variable name="v2" as="element()">
	     <b:complexMixedUserElem age="44" xsl:type="b:mixedPersonType">
         <description>Mr <name>Peter</name> has brown hair</description>
         <company>He works at a <name>Notorios</name> company</company>
      </b:complexMixedUserElem>
   </xsl:variable>

   <xsl:variable name="v3" as="element()">
	     <xsl:element name="c:myYear" type="c:yearType">2006</xsl:element>
   </xsl:variable>

   <xsl:template name="S1">
         <S1>
            <xsl:value-of select="$v1 instance of schema-element(a:complexTest)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$v1/@a:type instance of schema-attribute(a:type)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$v1"/>
         </S1>
   </xsl:template>
   
   <xsl:template name="S2">
         <S2>
            <xsl:value-of select="$v2 instance of element(*, b:mixedPersonType)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$v2 instance of schema-element(b:complexMixedUserElem)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$v2/*[1]"/>
         </S2>
   </xsl:template>
   
   <xsl:template name="S3">
         <S3>
            <xsl:value-of select="$v3 instance of element(*, c:yearType)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$v3 instance of schema-element(c:myYear)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$v3"/>
         </S3>
   </xsl:template>


</xsl:stylesheet>
