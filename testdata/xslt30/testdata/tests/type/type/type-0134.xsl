<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:xdt="http://www.w3.org/2005/02/xpath-datatypes" exclude-result-prefixes="xdt xs">

  <!-- PURPOSE: Test use of xs:untypedAtomic. -->
  <?spec xpath#id-predefined-types?>
  
  <xsl:param name="p" as="xs:date" select="current-date()"/>
  
  <xsl:template match="/">
    <out>;
        <xsl:call-template name="show-params">
          <xsl:with-param name="a" select="xs:untypedAtomic('FOO')"/>        
          <xsl:with-param name="b" select="xs:untypedAtomic('BAR')"/>
          <xsl:with-param name="c" select="xs:untypedAtomic('50')"/>          
        </xsl:call-template>                            
    </out> 
  </xsl:template>
  
  <xsl:template name="show-params">
     <xsl:param name="a" as="xs:string" select="'default'"/>
     <xsl:param name="b" as="xs:untypedAtomic"/>
     <xsl:param name="c" as="xs:double"/>
     <xsl:variable name="d" as="xs:untypedAtomic"><e>ORPHAN</e></xsl:variable>
     <q true="{$a instance of xs:string}"/>
     <q true="{$b instance of xs:untypedAtomic}"/>
     <q true="{$c instance of xs:double}"/>
     <q true="{$d instance of xs:untypedAtomic}"/>               
  </xsl:template>
  

</xsl:stylesheet>