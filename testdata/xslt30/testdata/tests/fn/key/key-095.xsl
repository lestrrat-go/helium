<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- Purpose: If there are several xsl:key declarations in the same package with the
    same key name, then they must all have the same effective value for their composite attribute.. -->

  <xsl:key name="mykey1" match="div" use="title,p[1]" composite="yes"/> 
  <xsl:key name="mykey1" match="span" use="title,p[1]"/> 

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="key('mykey1',('Expressions','Exp Section') )/q"/>      
    </out>
  </xsl:template>

</xsl:stylesheet>
