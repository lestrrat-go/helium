<?xml version="1.0"?> 

<!-- try024  -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
  exclude-result-prefixes="err">

  <!-- Test xsl:try 
  A xsl:result-document contains a xsl:try which contains a xsl:result-document, which
  also contains a xsl:try with a xsl:result-document, this fails and is catch by the inner xsl:try
  -->

  <xsl:template name="main">
    <xsl:param name="zero" select="0" as="xs:integer"/>
    <xsl:result-document href="try024.out">
      <xsl:try>
        <xsl:result-document href="try024_1.out">
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
    <xsl:param name="zero" select="0" as="xs:integer"/>
    <output>
    <xsl:try>
      <xsl:result-document href="try024_2.out">
        <xsl:for-each select="1 to 10">
          <value>
            <xsl:value-of select="position() div $zero"/>
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
