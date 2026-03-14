<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:one="http://ns.example.com/sch002"
				xmlns:two="http://ns.example.com/sch004"
				exclude-result-prefixes="one two"
				version="2.0">

  <!-- Purpose: to be imported into other stylesheets-->
 
<xsl:import-schema namespace="http://ns.example.com/sch004" schema-location="schema004.xsd"/>

<xsl:template name="imported">
  <xsl:text>&#10;</xsl:text>
  	<imported1>
    	<xsl:value-of select="$t1 instance of schema-element(one:complexTest)"/><xsl:text> * </xsl:text>
    	<xsl:value-of select="$t1/@one:type instance of schema-attribute(one:type)"/><xsl:text> * </xsl:text>
    	<xsl:copy-of select="$t1"/>
    </imported1>
    <xsl:text>&#10;</xsl:text>
    <imported2>
    	<xsl:value-of select="$t2 instance of element(*, two:mixedPersonType)"/><xsl:text> * </xsl:text>
    	<xsl:value-of select="$t2 instance of schema-element(two:complexMixedUserElem)"/><xsl:text> * </xsl:text>
    	<xsl:copy-of select="$t2/*[1]"/>
    </imported2>
    <xsl:text>&#10;</xsl:text>
</xsl:template>

<xsl:variable name="t1" as="element()">
    <one:complexTest xsl:type="one:complexTestType" one:type="333">
  	    <subject>math</subject>
  	    <size>50</size>
    </one:complexTest>
</xsl:variable>

<xsl:variable name="t2" as="element()">
	<two:complexMixedUserElem age="44" xsl:type="two:mixedPersonType">
		<description>Mr <name>Peter</name> has brown hair</description>
		<company>He works at a <name>Notorios</name> company</company>
	</two:complexMixedUserElem>
</xsl:variable>


</xsl:stylesheet>

