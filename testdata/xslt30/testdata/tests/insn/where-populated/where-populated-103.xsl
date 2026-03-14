<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <xsl:template name="xsl:initial-template">
       <out>
           <xsl:variable name="a" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="[]"/>
               </xsl:where-populated>
           </xsl:variable>
           <a count="{count($a)}"/>
           
           <xsl:variable name="b" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="[[]]"/>
               </xsl:where-populated>
           </xsl:variable>
           <b count="{count($b)}"/>
           
           <xsl:variable name="c" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="[()]"/>
               </xsl:where-populated>
           </xsl:variable>
           <c count="{count($b)}"/>
           
           <xsl:variable name="d" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="['']"/>
               </xsl:where-populated>
           </xsl:variable>
           <d count="{count($d)}"/>
           

           <xsl:variable name="empty" as="element(*)">
               <element/>
           </xsl:variable>
           <xsl:variable name="e" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="[$empty]"/>
               </xsl:where-populated>
           </xsl:variable>
           <e count="{count($e)}"/>
           
           <xsl:variable name="f" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="[[$empty, $empty], (), '']"/>
               </xsl:where-populated>
           </xsl:variable>
           <f count="{count($f)}"/>
           
           <xsl:variable name="z" as="item()*">
               <xsl:where-populated>
                   <xsl:sequence select="[42]"/>
               </xsl:where-populated>
           </xsl:variable>
           <z count="{count($z)}"/>
           
       </out>
    </xsl:template>
    
</xsl:stylesheet>