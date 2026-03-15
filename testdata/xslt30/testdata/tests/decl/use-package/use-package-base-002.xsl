<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-base-002"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  exclude-result-prefixes="xs p">
  
  <!-- tests that decimal-formats, keys, and namespace aliases are local to a package -->
  
 
  
    <xsl:decimal-format grouping-separator="~"/>
    <xsl:decimal-format name="a" grouping-separator="!"/>
  
    <xsl:function name="p:format" as="xs:string" visibility="public">
      <xsl:param name="in" as="xs:decimal"/>
      <xsl:value-of select="format-number($in, '0~000.0'), format-number($in, '0!000.0', 'a')"/>
    </xsl:function>
    
    <xsl:key name="k" match="row" use="column[1]"/>
    
    <xsl:function name="p:find" as="element(row)?" visibility="public">
      <xsl:param name="data" as="document-node()"/>
      <xsl:param name="search" as="xs:string"/>
      <xsl:sequence select="key('k', $search, $data)"/>
    </xsl:function>
    
    <xsl:namespace-alias stylesheet-prefix="xs" result-prefix="p"/>
    
    <xsl:function name="p:alias" as="element()" visibility="public">
      <xs:test/>
    </xsl:function>
    
    <xsl:output name="with-maps" use-character-maps="cm"/>
    <xsl:character-map name="cm">
      <xsl:output-character character="z" string="ZZ"/>
    </xsl:character-map>
    
    <xsl:template name="go" visibility="public">
      <xsl:result-document format="with-maps">
        <out>zzz</out>
      </xsl:result-document>
    </xsl:template>
    
  
  
</xsl:package>   