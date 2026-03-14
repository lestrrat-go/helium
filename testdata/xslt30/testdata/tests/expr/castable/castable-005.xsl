<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    version="3.0"
    xmlns:s="http://www.w3.org/XQueryTest/unionListDefined"
    exclude-result-prefixes="#all">
    
    <!-- castable operator with union types. Tests created by manual conversion of QT3 tests Castable-UnionType-* -->

    <xsl:import-schema namespace="http://www.w3.org/XQueryTest/unionListDefined" schema-location="unionListDefined.xsd"/>
    
    <xsl:function name="s:lazy">
        <xsl:param name="p"/>
        <xsl:sequence select="if (current-date() gt xs:date('1900-01-01')) then $p else '*!*!*!*'"/>
    </xsl:function>

    <xsl:variable name="qt3tests">
        <test-case name="Castable-UnionType-1" covers-30="cast-to-union">
            <description>Test the union type that should return an xs:integer value</description>
            <created by="O'Neil Delpratt" on="2011-10-27"/>
            <modified by="Michael Kay" on="2015-01-06" change="drop version declaration from query"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      "123" castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy("123") castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-2" covers-30="cast-to-union">
            <description>Test castability of xs:string to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      "2001-01-01" castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy("2001-01-01") castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-3" covers-30="cast-to-union">
            <description>Test castability of xs:string to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      "string" castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy("string") castable as s:myUnionType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-4" covers-30="cast-to-union">
            <description>Test castability of xs:untypedAtomic to union type.</description>
            <created by="O'Neil Delpratt" on="2011-10-27"/>
            <modified by="Michael Kay" on="2015-01-06" change="drop version declaration from query"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:untypedAtomic("123") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("123")) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-5" covers-30="cast-to-union">
            <description>Test castability of xs:untypedAtomic to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:untypedAtomic("2001-01-01") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("2001-01-01")) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-6" covers-30="cast-to-union">
            <description>Test castability of xs:untypedAtomic to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:untypedAtomic("string") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("string")) castable as s:myUnionType1'/></test2>           
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-7" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of one of the atomic types
                in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      s:unrestrictedInteger(123) castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:unrestrictedInteger(123)) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-8" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of one of the atomic types
                in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      s:unrestrictedDate("2001-01-01") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:unrestrictedDate("2001-01-01")) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-9" covers-30="cast-to-union">
            <description>Test castability of a value that is not an instance of one of the atomic
                types in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:gYear("2001") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:gYear("2001")) castable as s:myUnionType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-10" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of the union type
                itself</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      s:myUnionType1("2001-01-01") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:myUnionType1("2001-01-01")) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-11" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of one of the atomic types
                in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:integer(123) castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:integer(123)) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-12" covers-30="cast-to-union">
            <description>Test castability of a value that is castable to one or more of the atomic
                types in the transitive membership of the union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:date("2001-01-01") castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:date("2001-01-01")) castable as s:myUnionType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-13" covers-30="cast-to-union">
            <description>Test castability of empty sequence to a union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      () castable as s:myUnionType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(()) castable as s:myUnionType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-14" covers-30="cast-to-union">
            <description>Test the union type that should return an xs:integer value</description>
            <created by="O'Neil Delpratt" on="2011-10-27"/>
            <modified by="Michael Kay" on="2015-01-06" change="drop version declaration from query"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      "123" castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy("123") castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-15" covers-30="cast-to-union">
            <description>Test castability of xs:string to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      "2001-01-01" castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy("2001-01-01") castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-16" covers-30="cast-to-union">
            <description>Test castability of xs:string to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      "string" castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy("string") castable as s:myUnionType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-17" covers-30="cast-to-union">
            <description>Test castability of xs:untypedAtomic to union type.</description>
            <created by="O'Neil Delpratt" on="2011-10-27"/>
            <modified by="Michael Kay" on="2015-01-06" change="drop version declaration from query"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:untypedAtomic("123") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("123")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-18" covers-30="cast-to-union">
            <description>Test castability of xs:untypedAtomic to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:untypedAtomic("2001-01-01") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("123")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-19" covers-30="cast-to-union">
            <description>Test castability of xs:untypedAtomic to union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:untypedAtomic("string") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("string")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-20" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of one of the atomic types
                in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      s:unrestrictedInteger(123) castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:unrestrictedInteger(123)) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-21" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of one of the atomic types
                in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      s:unrestrictedDate("2001-01-01") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:unrestrictedDate("2001-01-01")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-22" covers-30="cast-to-union">
            <description>Test castability of a value that is not an instance of one of the atomic
                types in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:gYear("2001") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:gYear("2001")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-23" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of the union type
                itself</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      s:myUnionType1("2001-01-01") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:myUnionType1("2001-01-01")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-24" covers-30="cast-to-union">
            <description>Test castability of a value that is an instance of one of the atomic types
                in the transitive membership of the union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:integer(123) castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:integer(123)) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-25" covers-30="cast-to-union">
            <description>Test castability of a value that is castable to one or more of the atomic
                types in the transitive membership of the union type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      xs:date("2001-01-01") castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:date("2001-01-01")) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-26" covers-30="cast-to-union">
            <description>Test castability of empty sequence to a union type</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
      () castable as s:myUnionType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(()) castable as s:myUnionType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>



        <test-case name="Castable-UnionType-36" covers-30="type-derivation-with-unions">
            <description>Castable from a union type to string</description>
            <created by="Michael Kay" on="2014-05-04"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
         import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
         declare function local:f($a as s:dateOrDateTime) as xs:boolean { $a castable as xs:string };
         local:f(xs:date('2001-01-01'))
       ]]></test>
            <test2><xsl:value-of select='let $f := function ($a as s:dateOrDateTime) as xs:boolean { $a castable as xs:string }
                return $f(s:lazy(xs:date("2001-01-01")))'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-37" covers-30="type-derivation-with-unions">
            <description>Castable from a union type to one of the member types</description>
            <created by="Michael Kay" on="2014-05-04"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
         import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
         declare function local:f($a as s:dateOrDateTime) as xs:boolean { $a castable as xs:dateTime };
         local:f(xs:date('2001-01-01'))
       ]]></test>
            <test2><xsl:value-of select='let $f := function($a as s:dateOrDateTime) as xs:boolean { $a castable as xs:dateTime }
                return $f(s:lazy(xs:date("2001-01-01")))'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-UnionType-38" covers-30="type-derivation-with-unions">
            <description>Castable from a union type to one of the member types</description>
            <created by="Michael Kay" on="2014-05-04"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
         import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
         declare function local:f($a as s:dateOrDateTime) as xs:boolean { $a castable as xs:date };
         local:f(xs:date('2001-01-01'))
       ]]></test>
            <test2><xsl:value-of select='let $f := function ($a as s:dateOrDateTime) as xs:boolean { $a castable as xs:date }
                return $f(s:lazy(xs:date("2001-01-01")))'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>
    </xsl:variable>
    
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:for-each select="$qt3tests/test-case[(result/assert-true and not(test2 = 'true')) or (result/assert-false and not(test2 = 'false'))]">
                <failure test="{@name}"/>
            </xsl:for-each>
        </out>
    </xsl:template>

</xsl:stylesheet>
