<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.schemanodetest.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test schema-element($name) with an axis.-->

   <t:import-schema namespace="http://www.schemanodetest.example.com/ns/various"
                    schema-location="variousTypesNodeTest.xsd"/>

   <t:template match="/my:userNode">
	     <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:text>
</t:text>
         <out>
            <t:text>
</t:text>
            <E1>
               <t:value-of select="self::schema-element(my:userNode)/my:simpleUserList/name()"/>
            </E1>
            <t:text>
</t:text>
            <E2>
               <t:value-of select="descendant::schema-element(my:simpleUserList)"/>
            </E2>
            <t:text>
</t:text>
            <E3>
               <t:value-of select="descendant-or-self::schema-element(my:complexMixedUserElem)/*[1]"/>
            </E3>
            <t:text>
</t:text>
            <E4>
               <t:value-of select="child::schema-element(my:complexMixedUserElem)/child::description/child::name"/>
            </E4>
            <t:text>
</t:text>
            <t:apply-templates select="my:complexUserElem"/>
         </out>
         <t:text>
</t:text>
         <empty>
            <E12>
               <t:value-of select="attribute::schema-element(my:complexMixedUserElem)/*[1]"/>
            </E12>
            <E13>
               <t:value-of select="attribute::schema-element(my:complexMixedUserElem)/*[1]"/>
            </E13>
         </empty>
         <t:text>
</t:text>
      </output>
   </t:template>

   <t:template match="*">
	     <E5 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="parent::schema-element(my:userNode)/my:simpleUserList/name()"/>
      </E5>
      <t:text>
</t:text>
	     <E6 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="ancestor::schema-element(my:userNode)/my:simpleUserList"/>
      </E6>
      <t:text>
</t:text>
	     <E7 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="ancestor-or-self::schema-element(my:complexUserElem)/firstName"/>
      </E7>
      <t:text>
</t:text>
	     <E8 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="following::schema-element(my:complexUserElem)/firstName"/>
      </E8>
      <t:text>
</t:text>
	     <E9 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="following-sibling::schema-element(my:complexSimpleContentElem)"/>
      </E9>
      <t:text>
</t:text>
	     <E10 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="preceding::schema-element(my:complexMixedUserElem)/child::description/child::name"/>
      </E10>
      <t:text>
</t:text>
	     <E11 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="preceding-sibling::schema-element(my:complexMixedUserElem)/child::description/following-sibling::*"/>
      </E11>
      <t:text>
</t:text>
   </t:template>
</t:transform>
