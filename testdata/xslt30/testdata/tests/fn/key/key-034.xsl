<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- test key() in an xsl:number pattern, with a variable reference, 
     involves atomization, also part of a union pattern -->

  <xsl:key name="k1" match="*" use="Name/@First"/>
  <xsl:param name="name">
    <a>Ted</a>
  </xsl:param>

  <xsl:template match="/">
    <out>
      <xsl:for-each select="//Name">
        <p>
          <xsl:attribute name="number">
            <xsl:number level="any" count="key('k1', $name) | /"/>
          </xsl:attribute>
          <xsl:value-of select="."/>
        </p>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
