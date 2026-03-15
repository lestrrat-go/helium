<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://mhk015.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Test constructor for an xs:NOTATION subtype -->

<xsl:import-schema namespace="http://mhk015.uri/" schema-location="schema075.xsd"/>
<?spec fo#casting-to-derived-types?>



  <xsl:template match="/">
    <out>
      <n><xsl:value-of select="z:notat('z:de')"/></n>
    </out>
  </xsl:template>
  
</xsl:stylesheet>