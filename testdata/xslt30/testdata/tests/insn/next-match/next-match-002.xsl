<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test of next-match where the pattern of the dominant template is node() 
  				and it has @priority with value greater than the value of @priority on the secondary 
  				template which has a NameTest pattern. -->

   <t:template match="doc" priority="1">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:apply-templates/>
      </a>
   </t:template>

   <t:template match="foo">
      <c xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
   </t:template>

   <t:template match="node()" priority="2">
      <b xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </b>
   </t:template>
</t:transform>
