<?xml version="1.0" encoding="UTF-8"?>
<!-- Entry module for the staff-directory-html case.
     Exercises: xsl:import precedence with xsl:apply-imports (the overriding
     person template delegates its core rendering to the imported one), an
     xsl:key lookup, a param, and HTML output. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

  <xsl:import href="base.xsl"/>

  <xsl:output method="html" version="5.0" encoding="UTF-8" indent="no"/>

  <xsl:key name="dept-by-id" match="department" use="@id"/>

  <xsl:param name="orgName" as="xs:string" select="'Org'"/>

  <xsl:template match="/company">
    <html lang="en">
      <head>
        <meta charset="utf-8"/>
        <title><xsl:value-of select="$orgName"/></title>
      </head>
      <body>
        <h1><xsl:value-of select="$orgName"/></h1>
        <ul>
          <xsl:apply-templates select="staff/person"/>
        </ul>
      </body>
    </html>
  </xsl:template>

  <!-- Higher import precedence than base.xsl: wrap the base rendering and add
       the department name resolved via the key. -->
  <xsl:template match="person">
    <li>
      <xsl:apply-imports/>
      <span class="dept"><xsl:value-of select="key('dept-by-id', @dept)/@name"/></span>
    </li>
  </xsl:template>

</xsl:stylesheet>
