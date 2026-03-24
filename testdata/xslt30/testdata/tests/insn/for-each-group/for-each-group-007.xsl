<?spec xslt#grouping?>
<html xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<!-- test value-based grouping with explicit use of the codepoint collation -->
  <body>
    <xsl:for-each-group select="cities/city" group-by="substring(@name,1,1)"
                        collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
      <xsl:sort select="current-grouping-key()"/>
      <h2>
        <xsl:value-of select="upper-case(current-grouping-key())"/>
        <xsl:text> (</xsl:text>
        <xsl:value-of select="count(current-group())"/>
        <xsl:text>)</xsl:text>
      </h2>
      <xsl:for-each select="current-group()">
        <p><xsl:value-of select="@name"/></p>
      </xsl:for-each>
    </xsl:for-each-group>
  </body>
</html>