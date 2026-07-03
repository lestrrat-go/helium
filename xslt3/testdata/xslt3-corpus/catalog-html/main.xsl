<?xml version="1.0" encoding="UTF-8"?>
<!-- Entry module for the catalog-html case.
     Exercises: xsl:import (layout.xsl), xsl:include (format.xsl),
     named templates (xsl:call-template), two apply-templates modes
     (nav + body), several stylesheet params, and HTML output. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fmt="urn:helium:corpus:format"
    exclude-result-prefixes="xs fmt">

  <xsl:import href="layout.xsl"/>
  <xsl:include href="format.xsl"/>

  <xsl:output method="html" version="5.0" encoding="UTF-8" indent="no"/>

  <xsl:param name="siteTitle" as="xs:string" select="'Catalog'"/>
  <xsl:param name="currency" as="xs:string" select="'USD'"/>
  <xsl:param name="showSku" as="xs:boolean" select="false()"/>

  <xsl:template match="/catalog">
    <html lang="en">
      <head>
        <meta charset="utf-8"/>
        <title><xsl:value-of select="$siteTitle"/></title>
      </head>
      <body>
        <xsl:call-template name="page-header">
          <xsl:with-param name="title" select="$siteTitle"/>
        </xsl:call-template>
        <nav>
          <ul>
            <xsl:apply-templates select="category" mode="nav"/>
          </ul>
        </nav>
        <main>
          <xsl:apply-templates select="category" mode="body"/>
        </main>
        <xsl:call-template name="page-footer"/>
      </body>
    </html>
  </xsl:template>

  <xsl:template match="category" mode="nav">
    <li><a href="#{@id}"><xsl:value-of select="@name"/></a></li>
  </xsl:template>

  <xsl:template match="category" mode="body">
    <section id="{@id}">
      <h2><xsl:value-of select="@name"/></h2>
      <xsl:apply-templates select="product" mode="body"/>
    </section>
  </xsl:template>

  <xsl:template match="product" mode="body">
    <article>
      <h3><xsl:value-of select="name"/></h3>
      <xsl:if test="$showSku">
        <p class="sku">SKU: <xsl:value-of select="@sku"/></p>
      </xsl:if>
      <p class="price"><xsl:value-of select="fmt:money(price, $currency)"/></p>
    </article>
  </xsl:template>

</xsl:stylesheet>
