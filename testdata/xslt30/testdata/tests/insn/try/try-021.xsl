<?xml version="1.0"?> 

<!-- try022  -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
  exclude-result-prefixes="err xs">

  <!-- Test xsl:try creating a result document within the xsl:try; 
       fails on writing the second result document because the URI is the same -->

  <xsl:template name="main">
    <xsl:try>
      <xsl:for-each select="1 to 2">
        <xsl:result-document href="try022x.out" validation="strip">
          <output>
            <xsl:call-template name="test"/>
          </output>
        </xsl:result-document>
      </xsl:for-each>
      <xsl:catch errors="*">
        <err:error code="{$err:code}" module="{tokenize($err:module, '/')[last()]}" line="{$err:line-number}">
          <message>
            <xsl:value-of select="boolean($err:description)"/>
          </message>
        </err:error>
      </xsl:catch>
    </xsl:try>
  </xsl:template>


  <xsl:template name="test">
    <xsl:param name="two" select="2" as="xs:integer"/>
    <xsl:for-each select="1 to 10">
      <value>
        <xsl:value-of select="position() div $two"/>
      </value>
    </xsl:for-each>
  </xsl:template>


</xsl:stylesheet>
