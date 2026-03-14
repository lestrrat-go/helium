<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- XSLT 2.0: test use of select attribute of xsl:number -->

<xsl:template match="/">
  <out>
    [<xsl:number select="//note[.='ccc'][1]"/>]
    [<xsl:number select="//note[.='mmm'][1]" format="i"/>]
  </out>    
</xsl:template>


</xsl:stylesheet>
