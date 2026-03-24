<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- test xsl:number level="any" with a count pattern that matches the root of a non-document tree -->
<?spec xslt#number?>

  <xsl:template match="/">
    <xsl:variable name="test" as="element()">
      <a>
        <b/><b/><b/><c/>
      </a>
    </xsl:variable>
    <z><xsl:number level="any" select="$test/c" count="a|b|c"/></z>
  </xsl:template>


</xsl:stylesheet>
