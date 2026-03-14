<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- XSLT 2.0: parentless selected item for xsl:number -->

<xsl:template match="/">
  <xsl:variable name="n" as="element(*)"><a/></xsl:variable>
  <out>
    <xsl:number select="$n"/>
  </out>    
</xsl:template>


</xsl:stylesheet>
