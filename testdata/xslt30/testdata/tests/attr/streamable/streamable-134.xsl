<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:gml="http://www.opengis.net/gml"
	exclude-result-prefixes="xs gml"
	version="3.0">
	
	<xsl:mode streamable="yes"/>
	<xsl:output method="xml" indent="no"/>
	
	<xsl:template name="main">
	  <xsl:source-document streamable="true" href="transactions.xml">
        <xsl:apply-templates select="."/>
      </xsl:source-document>
	</xsl:template>
	
	<xsl:template match="account">
        <xsl:result-document href="{@nr}.xml">
          <root>
              <xsl:copy-of select="name()" />
          </root>
        </xsl:result-document>
	</xsl:template>
		
</xsl:stylesheet>
