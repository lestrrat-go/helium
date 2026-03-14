<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
  exclude-result-prefixes="xs math" 
  version="3.0">
  
  <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
  
  <xsl:mode _streamable="{$STREAMABLE}"/>
  
  <xsl:output indent="yes"/>
  
  <xsl:template match="tbody">
    <Entries>
      <xsl:for-each-group select="copy-of(tr)" group-starting-with="tr[th[@scope = 'rowgroup']]">
        <xsl:for-each select="current-group()">
          <Entry>
            <Year>
              <xsl:value-of select="substring(current-group()[1]/th[@scope = 'rowgroup'], 1, 4)"/>
            </Year>
            <Month>
              <xsl:value-of select="substring(current-group()[1]/th[@scope = 'rowgroup'], 5)"/>
            </Month>
            <Reason>
              <xsl:value-of select="th[@scope = 'row']"/>
            </Reason>
          </Entry>
        </xsl:for-each>
      </xsl:for-each-group>
    </Entries>
  </xsl:template>
  
</xsl:stylesheet>