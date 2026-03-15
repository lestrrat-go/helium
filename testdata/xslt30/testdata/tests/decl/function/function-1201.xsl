<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:saxon="http://saxon.sf.net/"
 xmlns:f="http://fxsl.sf.net/"
 exclude-result-prefixes="xs saxon f" 
 >
 
 <!-- bug reported by Dimitre Novatchev against Saxon 8.2 -->
 
 <xsl:variable name="root" select="/"/>
 
 <xsl:template match="/">
   <xsl:variable name="vSorted" as="node()+"
     select="f:sorted-nodelist(/*/book/@author)"/>
   <out>
   <xsl:for-each select="f:value-setS(/*/book/@author)">
     <author value="{@value}">
       <xsl:for-each select="
       f:value-setS(subsequence((: $vSorted, :)
                                f:sorted-nodelist($root/*/book/@author),
                                @start, 
                                @end - @start +1
                                )
                                 /../@title
                    )" >
          <title value="{@value}"/>
       </xsl:for-each>
     </author>
   </xsl:for-each>
   </out>
 </xsl:template>

 <xsl:function name="f:sorted-nodelist" as="node()*"
  saxon:memo-function="yes"
 >
 <!-- Note, saxon:memo-function is an optimization hint and should not affect the result -->
   <xsl:param name="pNodes" as="node()*"/>
   
   <xsl:for-each select="$pNodes">
     <xsl:sort/>
     <xsl:sequence select="."/>
   </xsl:for-each>
 </xsl:function>
 
 <xsl:function name="f:value-setS" as="node()*" saxon:memo-function="yes">
   <!-- Note, saxon:memo-function is an optimization hint and should not affect the result -->
   <xsl:param name="pNodes"  as="node()+"/>
   
   <xsl:variable name="vSorted" as="node()+"
     select="f:sorted-nodelist($pNodes)"/>
     
   <xsl:for-each select="distinct-values($pNodes)">
     <xsl:sort/>
     
     <xsl:variable name="vIndValues" as="xs:integer*"
      select="index-of($vSorted, .)"/>

     <set value="{.}" start="{$vIndValues[1]}" end="{$vIndValues[last()]}"/>
    
   </xsl:for-each>
 </xsl:function>
</xsl:stylesheet>
