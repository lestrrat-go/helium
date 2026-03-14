<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test the resolve-uri() function with two arguments -->


  <xsl:template match="doc" xml:base="http://no.other/home/">
    <out>
      <a>
        <xsl:value-of select="resolve-uri('', 'http://some.other/home/')"/>
      </a>
      <a>
        <xsl:value-of select="resolve-uri('banana.xml', 'http://some.other/home/')"/>
      </a>
      <a>
        <xsl:value-of select="resolve-uri('banana.xml', 'http://some.other/home')"/>
      </a>
      <a>
        <xsl:value-of select="resolve-uri('..', 'http://some.other/home/')"/>
      </a>
      <a>
        <xsl:value-of select="resolve-uri('http://www.wrox.com/', 'http://banana.com/')"/>
      </a>
    </out>
  </xsl:template>

</xsl:stylesheet>
