<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test promotion of independent subexpressions within a filter expression -->

 <xsl:param name="x" select="'111111'" 
    as="xs:string" xmlns:xs="http://www.w3.org/2001/XMLSchema"/>

  <xsl:template match="/">
    <out>
      <xsl:copy-of select="//section[@nr=number(substring($x, 1, 1))]" 
                     xmlns:saxon="http://saxon.sf.net/"/>
    </out>
  </xsl:template>

</xsl:stylesheet>