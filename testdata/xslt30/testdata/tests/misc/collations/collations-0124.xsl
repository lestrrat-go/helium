<!DOCTYPE xsl:stylesheet SYSTEM "collation.dtd">

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>
<xsl:variable name="x" as="xs:string" select="'Adele'"/>


<!-- general comparison using a case-blind collation as default collation, defined at various levels,
     result known at compile time -->

<xsl:key name="k" match="word" use="."/>

<xsl:template match="/" name="main">
<out>
  <xsl:call-template name="one"/>
  <xsl:call-template name="two"/>
  <xsl:call-template name="three"/>
</out>
</xsl:template>

  <xsl:template name="one" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive">
  <one><xsl:value-of select="$x = 'ADELE'"/></one>
</xsl:template>

<xsl:template name="two">
  <two xsl:default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"><xsl:value-of select="'Adele' = 'ADELE'"/></two>
</xsl:template>

<xsl:template name="three">
  <three><xsl:value-of select="$x = 'ADELE'" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/></three>
</xsl:template>

</xsl:stylesheet>

