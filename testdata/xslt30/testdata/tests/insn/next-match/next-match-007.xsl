<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="2.0">
  
  <!-- Purpose: Test apply-imports inside a template which pattern is a KindTest.The secondary 
    (imported) template has a NameTest pattern and hence higher priority. -->
  
  <xsl:import href="next-match-007a.xsl"/>
  
  <xsl:template match="element(*)">
    <a><xsl:apply-imports/></a>
  </xsl:template>
  
</xsl:stylesheet>
