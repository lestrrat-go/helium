<?xml version="1.0" encoding="UTF-8" ?>

<xsl:stylesheet version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:fn="http://www.w3.org/2005/xpath-functions" 
  xmlns:ecf2ncx="http://www.w3.org/2005/xpath-datatypes"  
  exclude-result-prefixes="xs ecf2ncx fn">
  
<xsl:function name="ecf2ncx:get-num-toc" as="xs:string?">
		<xsl:param name="e" as="element()"/>
		<xsl:variable name="name" select="local-name($e)"/>
		<!-- Renumérotation à partir d'une entrée hiérarchiquement plus importante -->
		<xsl:variable name="id" select="generate-id($e/parent::*)"/>
		<xsl:choose>
			<xsl:when test="$e/@toc-addnum='true' or $e/@toc-addnum='single'">
				<xsl:number level="single" select="$e"	count="*[generate-id(parent::*)=$id][local-name(.)=$name and @toc-addnum=$e/@toc-addnum]"
														format="{$e/@toc-addnum-format}"/>
				<!--BUG SAXON ? : on aurait pu écrire count="*[generate-id(parent::*)=$id][local-name(.)=$name][@toc-addnum=$e/@toc-addnum]" mais cela génère un bogue saxon-->
			</xsl:when>
			<xsl:when test="$e/@toc-addnum='any'">
				<!--@toc-addnum-format possibles sont "1", "01", "a", "A", "i", "I" (idem xsl:number/@format).-->
				<xsl:number level="any" select="$e"	count="*[ local-name() = $name and @toc-addnum = $e/@toc-addnum ]"
													format="{$e/@toc-addnum-format}"/>
			</xsl:when>
		</xsl:choose>  
</xsl:function>

<xsl:template name="main">
  <xsl:variable name="data">
    <e toc-addnum="single"/>
  </xsl:variable>
  <out><xsl:sequence select="ecf2ncx:get-num-toc($data/*)"/></out>
</xsl:template>
  
</xsl:stylesheet>
