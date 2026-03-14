<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xpath-default-namespace="http://www.tei-c.org/ns/1.0"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:mode on-no-match="shallow-copy" streamable="yes"/>
  
  <xsl:output method="html" indent="yes" html-version="5"/>
  
  <xsl:template match="/">
    <html>
      <head>
        <title>.NET XSLT Fiddle Example</title>
      </head>
      <body>
        <xsl:apply-templates select="TEI/text/body"/>
      </body>
    </html>
  </xsl:template>
  
  <xsl:template match="body">
    <div class="panel-body">
      <xsl:for-each-group select="*" group-starting-with="pb[position() mod 2 = 1]">
        <div style="display: flex; flex-direction: row;">
          <xsl:for-each-group select="current-group()" group-starting-with="pb">
            <div style="border: solid 1px red;">
              <xsl:apply-templates select="current-group()"/>
            </div>
          </xsl:for-each-group>
        </div>
      </xsl:for-each-group>
    </div>
  </xsl:template>
  
  <xsl:template match="pb">
    <div class="page number" style="text-align:right;">
      <xsl:text>[F.</xsl:text>
      <xsl:value-of select="@n"/>
      <xsl:text>]</xsl:text>
      <hr/>
    </div>
  </xsl:template>
  
  <xsl:template match="p">
    <p>
      <xsl:apply-templates/>
    </p>
  </xsl:template>
  
</xsl:stylesheet>