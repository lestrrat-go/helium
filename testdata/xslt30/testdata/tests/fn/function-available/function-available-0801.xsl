<?xml version="1.0"?> 

<?spec xslt#testing-function-availability?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:f = "http://f.uri/" exclude-result-prefixes="f">

<!-- test function-available() called dynamically from an expression
     evaluated lazily -->
     <?category function-available?>

<xsl:function name="f:selection">
    <xsl:param name="doc"/>
    <xsl:sequence select="$doc/name[function-available(.)]"/>
</xsl:function>

  <xsl:template match="/">
      <out>
        <xsl:value-of select="f:selection(doc)" separator=","/>
      </out>
  </xsl:template>


</xsl:stylesheet>
