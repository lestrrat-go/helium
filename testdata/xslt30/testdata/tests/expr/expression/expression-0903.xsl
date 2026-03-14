<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test simple mapping operator: rhs delivers atomic values -->
<?spec xpath#id-path-expressions?>

<xsl:template match="/">
  <out>
    <xsl:value-of select="//item/@*/(concat(name(), '=', .))"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
