<?xml version="1.0"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet version="2.0"
     xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
 
<!-- An opportunity for an optimizer to reorder predicates --> 

<xsl:template match="foo">
<out>
  <xsl:copy-of select="bar[not(following::thwunk/descendant::text())][@a='1'][number(@seq) lt 9]"/>
</out>
</xsl:template>

</xsl:stylesheet>
