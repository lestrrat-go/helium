<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.uri/"
  exclude-result-prefixes="my xs">


  <!-- Test for key() used to create hyperlinks in HTML output.
       Modified to remove dependency on generate-id -->

  <xsl:key name="titles" match="div" use="title"/>
  <xsl:key name="id" match="@id" use="."/>
  
  <xsl:strip-space elements="*"/>

  <xsl:template match="doc">
    <out>
      <xsl:for-each select="div">
        <xsl:text>&#xa;</xsl:text>
        <div id="{my:local-id(.)}">
          <xsl:number level="multiple" count="div" format="1.1. "/>
          <xsl:apply-templates/>
        </div>  
      </xsl:for-each>
    </out>
  </xsl:template>
  
  <xsl:template match="title">
    <title><xsl:value-of select="."/></title>
  </xsl:template>

  <xsl:template match="p">
    <P>
      <xsl:apply-templates/>
    </P>
  </xsl:template>

  <xsl:template match="divref">
    <A href="#{my:local-id(key('titles', .))}">
      <xsl:for-each select="key('titles', .)">
        <xsl:number level="multiple" count="div" format="1.1. "/>
      </xsl:for-each>
      <xsl:value-of select="."/>
    </A>
  </xsl:template>

  <xsl:template match="keydivref">
    <A href="#{my:local-id(key('id', .)/..)}">
      <xsl:for-each select="key('id', .)">KEY <xsl:number level="multiple" count="div"
          format="1.1. "/>
      </xsl:for-each>
      <xsl:value-of select="."/>
    </A>
  </xsl:template>

  <xsl:function name="my:local-id" as="xs:string">
    <xsl:param name="node" as="node()"/>
    <xsl:number select="$node" level="any" format="i"/>
  </xsl:function>

</xsl:stylesheet>
