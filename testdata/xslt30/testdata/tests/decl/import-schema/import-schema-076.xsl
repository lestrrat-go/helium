<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema068.uri/"
        xmlns:t="http://schema068.uri/"
        exclude-result-prefixes="xs t">

<!-- PURPOSE: Distinguish XS_ANY_TYPE from XS_UNTYPED -->
<?spec xslt#import-schema?>
<?spec xpath#id-instance-of?>

<xsl:import-schema schema-location="schema068.xsd" 
     namespace="http://schema068.uri/"/>
  



  <xsl:template match="/">
    <xsl:variable name="n">
    <out>
      <person id="p1">
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <person id="p2">
        <first>Penny</first>
        <surname>Whelan</surname>
      </person>
      <person id="p3">
        <last>Kay</last>
        <first>Michael</first>
        <middle>H</middle>
      </person> 
    </out>
    </xsl:variable>
    <xsl:variable name="v">
      <xsl:copy-of select="$n" validation="strict"/>
    </xsl:variable>    
    <xsl:variable name="o">
      <xsl:copy-of select="$v" validation="strip"/>
    </xsl:variable>
    <xsl:variable name="p">
      <xsl:copy-of select="$v" validation="preserve"/>
    </xsl:variable>
    <xsl:variable name="q">
      <xsl:copy-of select="$o" validation="preserve"/>
    </xsl:variable> 
    <xsl:variable name="r">
      <xsl:for-each select="$o/*">
        <xsl:copy validation="preserve">
          <xsl:copy-of select="*" validation="strip"/>
        </xsl:copy>
      </xsl:for-each>
    </xsl:variable>  
    <xsl:variable name="s">
      <xsl:for-each select="$o/*">
        <xsl:element name="s" validation="preserve">
          <xsl:copy-of select="*" validation="preserve"/>
        </xsl:element>
      </xsl:for-each>
    </xsl:variable>               
    <ans>
      <n is-untyped="{$n/* instance of element(*,xs:untyped)}" is-anyType="{$n/* instance of element(*,xs:anyType)}"/>    
      <o is-untyped="{$o/* instance of element(*,xs:untyped)}" is-anyType="{$o/* instance of element(*,xs:anyType)}"/>
      <p is-untyped="{$p/* instance of element(*,xs:untyped)}" is-anyType="{$p/* instance of element(*,xs:anyType)}"/> 
      <q is-untyped="{$q/* instance of element(*,xs:untyped)}" is-anyType="{$q/* instance of element(*,xs:anyType)}"/>  
      <r is-untyped="{$r/* instance of element(*,xs:untyped)}" is-anyType="{$r/* instance of element(*,xs:anyType)}"/> 
      <s is-untyped="{$s/* instance of element(*,xs:untyped)}" is-anyType="{$s/* instance of element(*,xs:anyType)}"/>                  
    </ans>       
  </xsl:template>
  
</xsl:stylesheet>