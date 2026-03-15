<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs"
  default-mode="m">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <!-- non-streamable use of current() in match pattern -->

  <xsl:mode name="m" streamable="yes" on-no-match="shallow-copy"/>
  
  <xsl:variable name="cat" select="'MMP'"/>

  <xsl:template name="xsl:initial-template">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:apply-templates/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <xsl:template match="ITEM[namespace-uri(current()) = '']">
    <xsl:next-match/>
  </xsl:template>

  <xsl:template match="AUTHOR[parent::ITEM[current()/data()='Jasper Fforde']]">
    <xsl:next-match/>
  </xsl:template>
  
  <xsl:template match="AUTHOR[current()/../@CAT='H']"/>
  
  <xsl:template match="AUTHOR[..[@CAT = current()/../@CAT]]">
    <xsl:next-match/>
  </xsl:template>
  
  <xsl:template match="*[ancestor::*[node-name(.) = node-name(current())]]" priority="100">
    <xsl:message terminate="yes">Recursive data found</xsl:message>
  </xsl:template>

</xsl:stylesheet>
