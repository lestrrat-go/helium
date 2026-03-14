<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:hat="http://hats.uri/"
        exclude-result-prefixes="xs hat">

<!-- PURPOSE: Test use of castable with user-defined date types. -->

  
<xsl:import-schema namespace="http://hats.uri/" schema-location="schema030.xsd"/>

  <xsl:template match="/">
    <out>
      <hatsize>
      <t><xsl:value-of select="'2003-02-28' castable as hat:date2003"/></t> 
      <f><xsl:value-of select="'2003-02-29' castable as hat:date2003"/></f> 
      <t><xsl:value-of select="'2003-12-31' castable as hat:date2003"/></t> 
      <f><xsl:value-of select="'2002-12-31' castable as hat:date2003"/></f> 
      <f><xsl:value-of select="'2004-01-01' castable as hat:date2003"/></f> 
      </hatsize>               
    </out>
  </xsl:template>
  
</xsl:stylesheet>