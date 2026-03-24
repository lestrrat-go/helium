<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test promotion of independent subexpressions within a filter expression -->

 <xsl:param name="x" select="1"/>

  <xsl:template match="/">
    <out>
      <xsl:copy-of select="//section[@nr=$x+1]" 
                     xmlns:saxon="http://saxon.sf.net/"/>
    </out>
  </xsl:template>

</xsl:stylesheet>