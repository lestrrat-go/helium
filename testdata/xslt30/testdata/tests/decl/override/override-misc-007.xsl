<xsl:package 
  name="http://www.w3.org/xslt30tests/override-misc-007"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-misc-002"
  exclude-result-prefixes="xs p">
  
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-misc-002"
    package-version="1.0.0"/>
    
  <!-- Streamed accumulators in different packages can share the same name -->
  
  <xsl:accumulator name="ac" initial-value="0" streamable="yes">
    <xsl:accumulator-rule match="*" select="$value - 1"></xsl:accumulator-rule>
  </xsl:accumulator>
  
  <xsl:template name="use-this" visibility="public">
    <xsl:source-document streamable="true" href="../accumulator/books.xml" use-accumulators="ac">
      <b><count nodes="{count(//*)}"/><acc value="{accumulator-after('ac')}"/></b>
    </xsl:source-document>  
  </xsl:template>
   
    <xsl:template name="main" visibility="public">
        <out>
          <xsl:call-template name="use-base"/>
          <xsl:call-template name="use-this"/>
        </out>   
    </xsl:template>
    
 
  
</xsl:package>   