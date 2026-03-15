<!-- the simplest possible package: unnamed mode can be the initial mode in any case -->
<xsl:package version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  declared-modes="false">
  

    <xsl:template match=".">
      <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   