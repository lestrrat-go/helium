<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:hat="http://hats.uri/"
        exclude-result-prefixes="xs hat">

<!-- PURPOSE: Test use of castable with user-defined pattern facet. -->
              <?spec xslt#import-schema?>
              <?spec xpath#id-castable?>
  
<xsl:import-schema namespace="http://hats.uri/" schema-location="schema030.xsd"/>

  <xsl:template match="/">
    <out>
      <hatsize>
      <t><xsl:value-of select="'abc123' castable as hat:vreg"/></t> 
      <t><xsl:value-of select="'xyz987' castable as hat:vreg"/></t> 
      <f><xsl:value-of select="'XYZ012' castable as hat:vreg"/></f> 
      <f><xsl:value-of select="'[abc123]' castable as hat:vreg"/></f> 
      <f><xsl:value-of select="' abc123 ' castable as hat:vreg"/></f> 
      </hatsize>               
    </out>
  </xsl:template>
  
</xsl:stylesheet>