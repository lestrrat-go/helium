<?xml version="1.0" encoding="UTF-8"?>
<!-- Entry module for the glossary-package case.
     Exercises: xsl:use-package (the package is served by name through the
     confined PackageResolver), a public xsl:function consumed from the used
     package, for-each-group + sort, a param, and XML output. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:tu="urn:helium:corpus:text-utils-fn"
    exclude-result-prefixes="xs tu">

  <xsl:use-package name="urn:helium:corpus:text-utils" package-version="1.0"/>

  <xsl:output method="xml" encoding="UTF-8" indent="yes"/>

  <xsl:param name="glossaryTitle" as="xs:string" select="'Glossary'"/>

  <xsl:template match="/glossary">
    <index title="{$glossaryTitle}">
      <xsl:for-each-group select="entry" group-by="tu:initial(term)">
        <xsl:sort select="current-grouping-key()"/>
        <group letter="{current-grouping-key()}">
          <xsl:for-each select="current-group()">
            <xsl:sort select="term"/>
            <term name="{term}"><xsl:value-of select="definition"/></term>
          </xsl:for-each>
        </group>
      </xsl:for-each-group>
    </index>
  </xsl:template>

</xsl:stylesheet>
