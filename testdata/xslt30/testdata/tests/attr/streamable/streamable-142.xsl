<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
  expand-text="true" version="3.0">
  
  <xsl:param name="STREAMABLE" as="xs:boolean" static="yes" select="true()"/>
  
  <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
  
  <xsl:output indent="yes"/>
  
  <xsl:template match="Record">
    <xsl:copy>
      <xsl:for-each-group select="tokenize(., '\r?\n')[normalize-space()]" 
        group-starting-with=".[starts-with(., ':A:')]">
        <Detail>
          <FieldA>{replace(., ':A:', '')}</FieldA>
          <Trans>
            <xsl:apply-templates select="current-group()[position() gt 1]"/>
          </Trans>
        </Detail>
      </xsl:for-each-group>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match=".[starts-with(., ':B:')]"> <!-- NOT  MOTIONLESS -->
    <Group>
      <FieldB>{replace(., ':B:', '')}</FieldB>
    </Group>
  </xsl:template>
  
</xsl:stylesheet>