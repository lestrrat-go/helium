<?xml version="1.0"?> 

<!-- try023  -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
  exclude-result-prefixes="err xs">

  <!-- Test xsl:try -->

  <xsl:template name="main">
    <xsl:param name="zero" select="0" as="xs:integer"/>
    <xsl:result-document href="try023.out">
      <xsl:try>
        <xsl:result-document href="try023_1.out">
          <output>
            <xsl:call-template name="InnerTryTest"/>
          </output>
        </xsl:result-document>

        <xsl:catch errors="*">
          <err:error code="{$err:code}" module="{$err:module}" line="{$err:line-number}">
            <message>
              <xsl:value-of select="$err:description"/>
            </message>
          </err:error>
        </xsl:catch>
      </xsl:try>
      <output>main document</output>
    </xsl:result-document>
  </xsl:template>


  <xsl:template name="InnerTryTest">
    <xsl:param name="two" select="2" as="xs:integer"/>
    <output>
    <xsl:try>
      <xsl:result-document href="try023_2.out">
        <xsl:for-each select="1 to 10">
          <value>
            <xsl:value-of select="position() div $two"/>
          </value>
        </xsl:for-each>
      </xsl:result-document>
      <xsl:catch errors="*">
        <err:error code="{$err:code}" module="{$err:module}" line="{$err:line-number}">
          <message>
            <xsl:value-of select="$err:description"/>
          </message>
        </err:error>
      </xsl:catch>
    </xsl:try>
      <note>secondary document written</note>
    </output>
  </xsl:template>


</xsl:stylesheet>
