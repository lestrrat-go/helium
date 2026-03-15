<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:hat="http://hats.uri/"
        exclude-result-prefixes="xs hat">

<!-- PURPOSE: Test creating an element node with a user-defined atomic type. -->
              <?spec xslt#import-schema?>
              <?spec xslt#annotation-for-constructed-element?>
              <?spec xpath#id-element-test?>
  
<xsl:import-schema namespace="http://hats.uri/" schema-location="schema030.xsd"/>

<xsl:param name="p" select="8"/>

  <xsl:template match="/">
    <out>
      <xsl:variable name="data" as="element(*,hat:hatsize)*">
        <xsl:element name="a" type="hat:bighatsize"><xsl:value-of select="$p"/></xsl:element>
        <xsl:element name="b" type="hat:bighatsize"><xsl:value-of select="$p+1"/></xsl:element>
        <xsl:element name="c" type="hat:hatsize"><xsl:value-of select="10"/></xsl:element>
      </xsl:variable>
      
      <hatsize>
        <xsl:value-of select="count($data[self::element(*,hat:hatsize)])"/>
        <xsl:text>,</xsl:text>
        <xsl:value-of select="count($data[self::element(*,hat:bighatsize)])"/>        
      </hatsize>               
    </out>
  </xsl:template>
  
</xsl:stylesheet>