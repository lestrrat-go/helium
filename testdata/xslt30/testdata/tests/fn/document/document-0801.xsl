<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test doc() function where the argument comes from a node, 
  				a variable and a subsequence. Relative URIs are resolved 
  				against the base-uri of the static context.-->

   <t:variable name="var" select="'doc05.xml'"/>
   
   <t:strip-space elements="*"/>

   <t:template match="defaultcontent" mode="a">
      <a>
         <t:copy-of select="doc(places[1])"/>
      </a>
   </t:template>
 
   <t:template match="defaultcontent" mode="b">
      <b>
         <t:copy-of select="doc($var)"/>
      </b>
   </t:template>
 
   <t:template match="defaultcontent" mode="c">
      <c>
         <t:copy-of select="doc(subsequence((places, $var, places[2]), 3,1))"/>
      </c>
   </t:template>
</t:transform>
