<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema061.uri/"
        xmlns:s61="http://schema061.uri/"
        exclude-result-prefixes="xs s61">

<!-- PURPOSE: Test "as" attributes on templates: simulate bug reported by Alan Painter -->
<?spec xslt#defining-templates?>

<xsl:import-schema schema-location="schema061.xsd" 
     namespace="http://schema061.uri/"/>
  



  <xsl:template match="/">
    <xsl:apply-templates select="doc"/>
  </xsl:template>
  
  <xsl:template match="element(doc, xs:untyped)" as="element(*, s61:personType)">
    <xsl:apply-templates select="current()" mode="B">
      <xsl:with-param name="elementName" select="'person'"/>
    </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="element(doc, xs:untyped)" mode="B" as="element(*, s61:personType)">
      <xsl:param name="elementName" as="xs:string" required="yes" />
      <xsl:element name="{$elementName}" type="s61:personType">
        <first>Michael</first>
        <middle>H</middle>
        <surname>Kay</surname>
      </xsl:element>
   </xsl:template>

</xsl:stylesheet>