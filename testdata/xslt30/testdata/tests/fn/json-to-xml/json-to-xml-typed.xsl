<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:j="http://www.w3.org/2005/xpath-functions">

    <!-- Tests for schema aware typing of nodes from json-to-xml()
    Note that these *only* check type and little more -
    the other non-typed tests should be employed for structure etc. 
    -->
    <xsl:param name="options.validate" select="map{'validate':true()}"/>

    <xsl:param name="s0.map" select="'{}'"/>
    <xsl:param name="s0.array" select="'[]'"/>
    <xsl:param name="s1.number" select="'[1]'"/>
    <xsl:param name="s1.string" select="'[&#34;a&#34;]'"/>
    <xsl:param name="s1.true" select="'[true]'"/>
    <xsl:param name="s1.false" select="'[false]'"/>
    <xsl:param name="s1.null" select="'[null]'"/>

    <xsl:import-schema namespace="http://www.w3.org/2005/xpath-functions"/>

    <!-- Note that whilst it would be cleaner perhaps to do the type checking in the assertions
        (though they are in another file :-( ) there appears to be no way for
        schema association with *result* documents, as opposed to source, 
        to be described in the XSLT test schema 
    
    Consequently we do the tests here and transmit true/false to the harness, in a very inelegant manner.
    
    When invocation of a function within XSLT is permitted, there will have to be some modifications I suspect
    -->
    <xsl:template name="json-to-xml-typed-001">
        <!-- <description>Basic map type</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" json-to-xml($s0.map,$options.validate)/* instance of element(j:map,j:mapType) "
            />
        </result>
    </xsl:template>
    <xsl:template name="json-to-xml-typed-002">
        <!-- <description>Basic array type</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" json-to-xml($s0.array,$options.validate)/* instance of element(j:array,j:arrayType) "
            />
        </result>
    </xsl:template>

    <xsl:template name="json-to-xml-typed-003">
        <!--<description>Array Number typed</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" let $result := json-to-xml($s1.number,$options.validate)/* return
            count($result) = 1 and
            $result instance of element(j:array) and
            $result/*[1] instance of element(j:number,j:numberType) and
            xs:double($result/*[1]) eq 1.0"
            />
        </result>
    </xsl:template>

    <xsl:template name="json-to-xml-typed-004">
        <!--<description>Array String typed</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" let $result := json-to-xml($s1.string,$options.validate)/* return
            count($result) = 1 and
            $result instance of element(j:array) and
            $result/*[1] instance of element(j:string,j:stringType) and
            xs:string($result/*[1]) eq 'a'"
            />
        </result>
    </xsl:template>
    <xsl:template name="json-to-xml-typed-005">
        <!--<description>Array True typed</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" let $result := json-to-xml($s1.true,$options.validate)/* return
            count($result) = 1 and
            $result instance of element(j:array) and
            $result/*[1] instance of element(j:boolean,xs:boolean) and
            xs:boolean($result/*[1]) eq true()"
            />
        </result>
    </xsl:template>
    <xsl:template name="json-to-xml-typed-006">
        <!--<description>Array False typed</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" let $result := json-to-xml($s1.false,$options.validate)/* return
            count($result) = 1 and
            $result instance of element(j:array) and
            $result/*[1] instance of element(j:boolean,xs:boolean) and
            xs:boolean($result/*[1]) eq false()"
            />
        </result>
    </xsl:template>
    <xsl:template name="json-to-xml-typed-007">
        <!--<description>Array Null typed</description>
        <created by="John Lumley" on="2013-10-08"/>
        <environment ref="JSON-XML-typed"/>-->
        <result>
            <xsl:value-of
                select=" let $result := json-to-xml($s1.null,$options.validate)/* return
            count($result) = 1 and
            $result instance of element(j:array) and
            $result/*[1] instance of element(j:null,j:nullType) and
            empty($result/*[1]/(*|text()))"
            />
        </result>
    </xsl:template>



</xsl:stylesheet>
