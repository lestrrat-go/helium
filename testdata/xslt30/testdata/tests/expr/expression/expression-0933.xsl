<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test simple mapping operator: type of rhs not known statically, delivers nodes then atoms at run-time -->
<?spec xpath#id-path-expressions?>
<?error XPTY0018?>


<xsl:param name="p" select="false()"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="//item/(if (position()=1) then @* else string(@val))"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
