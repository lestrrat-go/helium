<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema061.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Type-checking of constructed elements involving substitution groups -->
             <?spec xslt#annotation-for-constructed-element?>
              <?error ?>

<xsl:import-schema namespace="http://schema061.uri/" schema-location="schema061.xsd"/>



  <xsl:template match="/">
    <out>
      <xsl:call-template name="p"/>
    </out>
  </xsl:template>
  
  <xsl:template name="p" as="schema-element(z:person)">
    <xsl:element name="{$param}" namespace="http://schema061.uri/" type="xs:integer">32</xsl:element>
  </xsl:template>
  
  <xsl:param name="param" select="'person'"/>
  
</xsl:stylesheet>