<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0">

  <!-- FileName: position83 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.2 -->
  <!-- Purpose: Try count() and last() on unionpaths that aren't sequential. -->
  
<xsl:template match="/">
  <out>
    <xsl:for-each select="//section/title">
      Title= <xsl:value-of select="."/>
      <name>
        <xsl:attribute name="last-of">
          <xsl:value-of select="count(ancestor::section
            |ancestor::simplesect|ancestor::article)"/>
        </xsl:attribute>
        <xsl:value-of select="name((ancestor::section|ancestor::simplesect
          |ancestor::article)[last()])"/>
      </name>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
