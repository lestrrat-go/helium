<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="http://example.com/expression-1904" exclude-result-prefixes="f">

  <!-- PURPOSE test unparsed-entity-uri#2 function -->
<?spec xslt#unparsed-entity-uri?>
  <xsl:template match="doc">
    <out>
      <xsl:value-of select="f:f(.)"/>
    </out>
  </xsl:template>
  
    <xsl:function name="f:f">
    <xsl:param name="doc"/>
    <xsl:value-of select="ends-with(unparsed-entity-uri('hatch-pic', $doc), '/fn/grafix/OpenHatch.gif')"/>
   </xsl:function>         
 
 
</xsl:stylesheet>
