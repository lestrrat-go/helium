<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:gml="http://www.opengis.net/gml"
	exclude-result-prefixes="xs gml"
	version="3.0">
	
	<!-- Use has-children() in a predicate, proving it is motionless -->
	<xsl:mode streamable="yes"/>
	<xsl:output method="xml" indent="no"/>
	
	<xsl:template name="main">
	  <xsl:source-document streamable="true" href="transactions.xml">
	    <xsl:apply-templates select="."/>
	  </xsl:source-document>
	</xsl:template>
	
	<xsl:template match="/">
        <xsl:apply-templates/>
	</xsl:template>
	
	<xsl:template match="*[has-children()]">
      <xsl:copy>
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates/>
      </xsl:copy>
	</xsl:template>
	
	<xsl:template match="*[not(has-children())]">
      <xsl:copy>
        <xsl:copy-of select="@*"/>
        <xsl:attribute name="empty" select="true()"/>
      </xsl:copy>
	</xsl:template>
		
</xsl:stylesheet>
