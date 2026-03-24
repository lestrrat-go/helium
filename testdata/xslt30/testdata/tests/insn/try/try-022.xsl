<?xml version="1.0"?> 

<!-- try-022  -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
  exclude-result-prefixes="err xs">

  <!-- Test xsl:try -->

  <xsl:template name="main">
    <xsl:param name="zero" select="0" as="xs:integer"/>
    <xsl:result-document href="">
      <result>
        <xsl:try>
          <xsl:for-each select="1 to 2">
            <xsl:result-document href="try-022-{position()}.out">
              <output>
                <xsl:call-template name="test"/>
                <xsl:if test="position() = 2">
                  <xsl:value-of select="position() div $zero"/>
                </xsl:if>
              </output>
            </xsl:result-document>
          </xsl:for-each>
          <xsl:catch errors="*">
            <err:error code="{$err:code}" module="{tokenize($err:module, '/')[last()]}" line="{$err:line-number}">
              <message>
                <xsl:value-of select="$err:description"/>
              </message>
            </err:error>
          </xsl:catch>
        </xsl:try>
        <output>main document</output>
      </result>  
    </xsl:result-document>
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
