<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with matches($input?,$pattern) on a literal string where pattern uses back-references. Verifies that back-references match the 
  				exact string that was already matched by the capturing subexpressions they refer to and not just its pattern. Includes case where $input 
  				is an empty string or an empty sequence as well as cases where second digit of back-reference is or is not considered to be part of it 
  				because of sufficient/insufficient number of capturing substrings.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:value-of select="matches('kikikeriki!! Tak, tak, tak! - da kommen sie.','(da)( )(kommen)\2(sie)')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('kikikeriki!! Tak, tak, tak! - da kommen sie.','(ki){2}ke.*\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('kikikeriki!! Tak, tak, tak! - da kommen sie.','(ki|ke)\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('kikikeriki!! Tak, tak, tak! - da kommen sie.','(!).*\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('kikikeriki2!! Tak, tak, tak! - da kommen sie.','(ki){2}ke.*\12')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('','()\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches((),'()\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('123kikikeriki123','([0-9]+)([a-z]+)\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('123kikikeriki4566','(1)(2)(3)(ki)(k)(i)(ke)(ri)(ki)(4)(5)(6)\12')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('kikikeriki!! Tak, tak, tak! - da kommen sie.','(da)( )(kommen)\2(die)')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('kikikeriki!! Tak, tak, tak! - da kommen sie.','(ki){2}ke.*\12')"/>
         <t:text>, </t:text>
         <t:value-of select="matches('123kikikeriki456','([0-9]+)([a-z]+)\1')"/>
         <t:text>, </t:text>
         <t:value-of select="matches(' ','( )\1')"/>
      </out>
   </t:template>
</t:transform>
