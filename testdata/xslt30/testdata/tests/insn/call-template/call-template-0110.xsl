<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
	<!-- Purpose: Within a named template, we don't know statically whether there will be a context item.
	     This test tests various situations where a dynamic test for existence of a context item is needed. -->

	<xsl:template name="xsl:initial-template">
		<out>
			<!-- run with no context item -->
			<xsl:call-template name="all">
				<xsl:with-param name="context-exists" select="false()" tunnel="yes"/>
			</xsl:call-template>
			<xsl:variable name="doc" as="element()">
				<input status="ok"><z>ok</z></input>
			</xsl:variable>
			<xsl:for-each select="$doc">
				<!-- run with context item -->
				<xsl:call-template name="all">
					<xsl:with-param name="context-exists" select="true()" tunnel="yes"/>
				</xsl:call-template>
			</xsl:for-each>
		</out>
	</xsl:template>
	
	<xsl:template name="all">
		<xsl:call-template name="getAttribute"/>
		<xsl:call-template name="getChild"/>
	</xsl:template>

	<xsl:template name="getAttribute">
		<xsl:variable name="caught">
			<xsl:try select="data(@status)">
				<xsl:catch errors="*:XPDY0002" select="'caught'"/>
			</xsl:try>
		</xsl:variable>
		<getAtt status="{$caught}"/>
	</xsl:template>
	
	<xsl:template name="getChild">
		<xsl:variable name="caught">
			<xsl:try select="string(child::z)">
				<xsl:catch errors="*:XPDY0002" select="'caught'"/>
			</xsl:try>
		</xsl:variable>
		<getChild status="{$caught}"/>
	</xsl:template>
</xsl:transform>
