<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:my="http://www.example.com/ns/match/id-idref-notation"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="3.0">
   <!-- Purpose: parenthesized pattern does not match parentless node and has higher priority -->

   <xsl:variable name="data" as="element()">
     <a nr="1">
       <a nr="2"/>
     </a>
   </xsl:variable>    

   <xsl:template match="/">
      <out>
         <xsl:apply-templates select="$data"/>
      </out>
   </xsl:template>
   
   <xsl:template match="(: child-or-top:: :)a">
      <one nr="{@nr}">
        <xsl:apply-templates/>
      </one>  
   </xsl:template>

   <xsl:template match="(: child-or-top:: :)a">
      <two nr="{@nr}">
        <xsl:next-match/>
      </two>  
   </xsl:template>
   
   <xsl:template match="( */(: child:: :) a)">
      <three nr="{@nr}">
        <xsl:next-match/>
      </three>  
   </xsl:template>
   
   <xsl:template match="//a">
     <wrong reason="//a should only match a node in a tree rooted at a document node"/>
   </xsl:template>  

</xsl:transform>
