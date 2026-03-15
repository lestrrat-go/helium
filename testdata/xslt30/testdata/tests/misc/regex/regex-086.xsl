<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with matches($input?,$pattern) where $input comes from a variable, node, 
  				context node(.) and subsequence; pattern uses reluctant quantifiers.-->


   <t:variable name="v1" select="/doc/str1"/>

   <t:variable name="v2" select="'a'"/>

   <t:variable name="v3" select="''"/>

   <t:template match="/doc/str1" mode="a">       
         <out>
            <A><t:value-of select="matches(//str1,'a??')"/></A>
            <B><t:value-of select="matches(//str1,'a*?')"/></B>
            <C><t:value-of select="matches(//str1,'a+?')"/></C>
            <D><t:value-of select="matches(//str1,'a{2}?')"/></D>
            <E><t:value-of select="matches(//str1,'a{2,}?')"/></E>
            <F><t:value-of select="matches(//str1,'a{2,3}?')"/></F>
         </out>
   </t:template>  
   
   <t:template match="/doc/str1" mode="b">       
      <out>
         <A><t:value-of select="matches(.,'a??')"/></A>
         <B><t:value-of select="matches(.,'a*?')"/></B>
         <C><t:value-of select="matches(.,'a+?')"/></C>
         <D><t:value-of select="matches(.,'a{2}?')"/></D>
         <E><t:value-of select="matches(.,'a{2,}?')"/></E>
         <F><t:value-of select="matches(.,'a{2,3}?')"/></F>
      </out>
   </t:template>  
         
   <t:template match="/doc/str1" mode="c">       
      <out>
         <A><t:value-of select="matches($v1,'a??')"/></A>
         <B><t:value-of select="matches($v1,'a*?')"/></B>
         <C><t:value-of select="matches($v1,'a+?')"/></C>
         <D><t:value-of select="matches($v1,'a{2}?')"/></D>
         <E><t:value-of select="matches($v1,'a{2,}?')"/></E>
         <F><t:value-of select="matches($v1,'a{2,3}?')"/></F>
         <G><t:value-of select="matches($v2,'a{2,3}?')"/></G>
         <H><t:value-of select="matches($v3,'a*?')"/></H>
      </out>
   </t:template>
   
   <t:template match="/doc/str1" mode="d">       
      <out>
         <A><t:value-of select="matches(subsequence(('a','abraacadabra',''),2,1),'a??')"/></A>
         <B><t:value-of select="matches(subsequence(('a','abraacadabra',''),2,1),'a*?')"/></B>
         <C><t:value-of select="matches(subsequence(('a','abraacadabra',''),2,1),'a+?')"/></C>
         <D><t:value-of select="matches(subsequence(('a','abraacadabra',''),2,1),'a{2}?')"/></D>
         <E><t:value-of select="matches(subsequence(('a','abraacadabra',''),2,1),'a{2,}?')"/></E>
         <F><t:value-of select="matches(subsequence(('a','abraacadabra',''),2,1),'a{2,3}?')"/></F>
      </out>
   </t:template> 
         
</t:transform>
