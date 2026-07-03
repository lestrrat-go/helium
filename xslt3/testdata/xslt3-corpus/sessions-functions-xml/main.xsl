<?xml version="1.0" encoding="UTF-8"?>
<!-- Entry module for the sessions-functions-xml case.
     Exercises: xsl:include of a module that defines namespaced xsl:functions,
     a named template with a required typed param, params, and XML output. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:u="urn:helium:corpus:util"
    exclude-result-prefixes="xs u">

  <xsl:include href="util.xsl"/>

  <xsl:output method="xml" encoding="UTF-8" indent="yes"/>

  <xsl:param name="conferenceName" as="xs:string" select="'HeliumConf'"/>
  <xsl:param name="year" as="xs:integer" select="2026"/>

  <xsl:template match="/schedule">
    <program conference="{$conferenceName}" year="{$year}">
      <xsl:apply-templates select="session"/>
    </program>
  </xsl:template>

  <xsl:template match="session">
    <talk id="{u:slug(title)}" minutes="{u:duration(@start, @end)}">
      <title><xsl:value-of select="title"/></title>
      <xsl:for-each select="speaker">
        <xsl:call-template name="emit-speaker">
          <xsl:with-param name="who" select="."/>
        </xsl:call-template>
      </xsl:for-each>
    </talk>
  </xsl:template>

  <xsl:template name="emit-speaker">
    <xsl:param name="who" as="element(speaker)" required="yes"/>
    <presenter name="{$who}" affiliation="{$who/@org}"/>
  </xsl:template>

</xsl:stylesheet>
