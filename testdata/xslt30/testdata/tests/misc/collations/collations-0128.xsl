<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"
  exclude-result-prefixes="xs">

  <xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string"
    select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>
  <xsl:param name="x" as="xs:string" select="'Adele'"/>
  <xsl:param name="expr" as="xs:string">. = 'ADELE'</xsl:param>


  <!-- general comparison using a case-blind collation as default collation, defined at various levels -->
  <!-- variant of collation-0126 in which the default collation is needed dynamically (in xsl:evaluate) -->

  <xsl:key name="k" match="word" use="."/>

  <xsl:template match="/" name="main">
    <out>
      <xsl:call-template name="one"/>
      <xsl:call-template name="two"/>
      <xsl:call-template name="three"/>
    </out>
  </xsl:template>

  <xsl:template name="one">
    <one>
      <xsl:evaluate xpath="$expr" context-item="$x"/>
    </one>
  </xsl:template>

  <xsl:template name="two" default-collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
    <two xsl:default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive">
      <xsl:evaluate xpath="$expr" context-item="$x"/>
    </two>
  </xsl:template>

  <xsl:template name="three" default-collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
    <three>
      <xsl:evaluate xpath="$expr" context-item="$x" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/>
    </three>
  </xsl:template>

</xsl:stylesheet>
