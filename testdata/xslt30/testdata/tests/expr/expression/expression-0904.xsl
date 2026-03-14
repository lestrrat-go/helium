<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test simple mapping operator: type of rhs not known statically, delivers nodes at run-time -->
<?spec xpath#id-path-expressions?>

<xsl:param name="p" select="true()"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="//item/@*/(if ($p) then . else string(.))"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
