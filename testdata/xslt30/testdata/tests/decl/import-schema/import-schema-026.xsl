<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:hat="http://hats.uri/"
        exclude-result-prefixes="xs hat">

<!-- PURPOSE: Test creating an attribute node with a user-defined atomic type. -->
              <?spec xslt#import-schema?>
              <?spec xslt#annotation-for-constructed-attribute?>
              <?spec xpath#id-attribute-test?>
  
<xsl:import-schema namespace="http://hats.uri/" schema-location="schema030.xsd"/>

<xsl:param name="p" select="8"/>

  <xsl:template match="/">
    <out>
      <xsl:variable name="data" as="attribute(*,hat:hatsize)*">
        <xsl:attribute name="a" select="$p" type="hat:bighatsize"/>
        <xsl:attribute name="b" select="$p+1" type="hat:bighatsize"/>
        <xsl:attribute name="c" select="10" type="hat:hatsize"/>
      </xsl:variable>
      
      <hatsize>
        <xsl:value-of select="count($data[self::attribute(*,hat:hatsize)])"/>
        <xsl:text>,</xsl:text>
        <xsl:value-of select="count($data[self::attribute(*,hat:bighatsize)])"/>        
      </hatsize>               
    </out>
  </xsl:template>
  
</xsl:stylesheet>